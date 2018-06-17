package alicloud

import (
	"fmt"
	"strings"

	"github.com/aliyun/aliyun-log-go-sdk"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAlicloudLogStoreIndex() *schema.Resource {
	return &schema.Resource{
		Create: resourceAlicloudLogStoreIndexCreate,
		Read:   resourceAlicloudLogStoreIndexRead,
		Update: resourceAlicloudLogStoreIndexUpdate,
		Delete: resourceAlicloudLogStoreIndexDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"project": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"logstore": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"index_type": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Default:      FullText,
				ValidateFunc: validateAllowedStringValue([]string{string(FullText), string(Field)}),
			},
			"case_sensitive": &schema.Schema{
				Type:             schema.TypeBool,
				Optional:         true,
				Default:          false,
				DiffSuppressFunc: logStoreIndexFieldNumberFieldDiffSuppressFunc,
			},
			"include_chinese": &schema.Schema{
				Type:             schema.TypeBool,
				Optional:         true,
				Default:          false,
				DiffSuppressFunc: logStoreIndexFieldNumberFieldDiffSuppressFunc,
			},
			"token": &schema.Schema{
				Type:             schema.TypeString,
				Optional:         true,
				DiffSuppressFunc: logStoreIndexTokenFieldDiffSuppressFunc,
			},
			//field search
			"field_name": &schema.Schema{
				Type:             schema.TypeString,
				Optional:         true,
				DiffSuppressFunc: logStoreIndexFieldDiffSuppressFunc,
			},
			"field_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  LongType,
				ValidateFunc: validateAllowedStringValue([]string{string(TextType), string(LongType),
					string(DoubleType), string(JsonType)}),
				DiffSuppressFunc: logStoreIndexFieldDiffSuppressFunc,
			},
			"field_alias": &schema.Schema{
				Type:             schema.TypeString,
				Optional:         true,
				DiffSuppressFunc: logStoreIndexFieldDiffSuppressFunc,
			},
			"enable_analytics": &schema.Schema{
				Type:             schema.TypeBool,
				Optional:         true,
				Default:          false,
				DiffSuppressFunc: logStoreIndexFieldDiffSuppressFunc,
			},
		},
	}
}

func resourceAlicloudLogStoreIndexCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AliyunClient)

	project := d.Get("project").(string)
	store, err := client.DescribeLogStore(project, d.Get("logstore").(string))
	if err != nil {
		return fmt.Errorf("DescribeLogStore got an error: %#v.", err)
	}

	new, err := buildLogIndex(d, IndexType(d.Get("index_type").(string)), store)
	if err != nil {
		return err
	}

	old, err := store.GetIndex()
	if err != nil && !IsExceptedErrors(err, []string{IndexConfigNotExist}) {
		return fmt.Errorf("While Creating index, GetIndex got an error: %#v.", err)
	}
	index_type := IndexType(d.Get("index_type").(string))
	if old != nil {
		line := old.Line
		keys := old.Keys
		if index_type == FullText && line != nil {
			return fmt.Errorf("There is aleady existing a %s index in the store %s. Please import it using id '%s%s%s'.",
				index_type, store.Name, project, COLON_SEPARATED, store.Name)
		}
		if _, ok := keys[d.Get("field_name").(string)]; index_type == Field && ok {
			return fmt.Errorf("There is aleady existing a %s index with key %s in the store %s. Please import it using id '%s%s%s%s%s'.",
				index_type, d.Get("field_name").(string), store.Name, project, COLON_SEPARATED, store.Name, COLON_SEPARATED, d.Get("field_name").(string))
		}
		if err := store.UpdateIndex(new); err != nil {
			return fmt.Errorf("UpdateLogStoreIndex got an error: %#v.", err)
		}
	} else {
		if err := store.CreateIndex(new); err != nil {
			return fmt.Errorf("CreateLogStoreIndex got an error: %#v.", err)
		}
	}
	id := fmt.Sprintf("%s%s%s", project, COLON_SEPARATED, store.Name)

	if index_type == Field {
		id = fmt.Sprintf("%s%s%s", id, COLON_SEPARATED, d.Get("field_name").(string))
	}

	d.SetId(id)

	return resourceAlicloudLogStoreIndexUpdate(d, meta)
}

func resourceAlicloudLogStoreIndexRead(d *schema.ResourceData, meta interface{}) error {
	split := strings.Split(d.Id(), COLON_SEPARATED)

	index, err := meta.(*AliyunClient).DescribeLogStoreIndex(split[0], split[1])

	if err != nil {
		if NotFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("GetIndex got an error: %#v.", err)
	}

	if len(split) == 2 {
		if index.Line == nil {
			d.SetId("")
			return nil
		}
		d.Set("index_type", FullText)
		d.Set("case_sensitive", index.Line.CaseSensitive)
		d.Set("include_chinese", index.Line.Chn)
		d.Set("token", strings.Join(index.Line.Token, ""))
	} else {
		value, ok := index.Keys[split[2]]
		if !ok {
			d.SetId("")
			return nil
		}
		d.Set("index_type", Field)
		d.Set("field_name", split[2])
		d.Set("case_sensitive", value.CaseSensitive)
		d.Set("include_chinese", value.Chn)
		d.Set("token", strings.Join(value.Token, ""))
		d.Set("field_type", value.Type)
		d.Set("enable_analytics", value.DocValue)
	}

	d.Set("project", split[0])
	d.Set("logstore", split[1])

	return nil
}

func resourceAlicloudLogStoreIndexUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AliyunClient)

	if d.IsNewResource() {
		return resourceAlicloudLogStoreIndexRead(d, meta)
	}

	split := strings.Split(d.Id(), COLON_SEPARATED)
	d.Partial(true)

	update := false
	if d.HasChange("case_sensitive") {
		update = true
		d.SetPartial("case_sensitive")
	}
	if d.HasChange("include_chinese") {
		update = true
		d.SetPartial("include_chinese")
	}
	if d.HasChange("token") {
		update = true
		d.SetPartial("token")
	}

	if len(split) > 2 {
		if d.HasChange("field_name") {
			update = true
			d.SetPartial("field_name")
		}
		if d.HasChange("field_type") {
			update = true
			d.SetPartial("field_type")
		}
		if d.HasChange("field_alias") {
			update = true
			d.SetPartial("field_alias")
		}
		if d.HasChange("enable_analytics") {
			update = true
			d.SetPartial("enable_analytics")
		}
	}

	if update {

		store, err := client.DescribeLogStore(split[0], split[1])
		if err != nil {
			return fmt.Errorf("DescribeLogStore got an error: %#v.", err)
		}

		indexType := FullText
		if len(split) > 2 {
			indexType = Field
		}
		index, err := buildLogIndex(d, indexType, store)
		if err != nil {
			return err
		}

		if err := store.UpdateIndex(index); err != nil {
			return fmt.Errorf("UpdateLogStoreIndex got an error: %#v.", err)
		}
	}
	d.Partial(false)

	return resourceAlicloudLogStoreIndexRead(d, meta)
}

func resourceAlicloudLogStoreIndexDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AliyunClient)

	split := strings.Split(d.Id(), COLON_SEPARATED)

	index, err := client.DescribeLogStoreIndex(split[0], split[1])
	if err != nil {
		if NotFoundError(err) {
			return nil
		}
		return fmt.Errorf("While deleting index, GetIndex got an error: %#v.", err)
	}

	if len(split) == 2 {
		if index.Line == nil {
			d.SetId("")
			return nil
		}
		index.Line = nil
	} else {
		if _, ok := index.Keys[split[2]]; !ok {
			d.SetId("")
			return nil
		}
		delete(index.Keys, split[2])
	}

	if index.Line == nil && len(index.Keys) < 1 {
		if err := client.logconn.DeleteIndex(split[0], split[1]); err != nil {
			return fmt.Errorf("DeleteIndex got an error: %#v.", err)
		}
	}
	if err := client.logconn.UpdateIndex(split[0], split[1], *index); err != nil {
		return fmt.Errorf("UpdateIndex got an error: %#v.", err)
	}
	return nil
}

func buildLogIndex(d *schema.ResourceData, indexType IndexType, store *sls.LogStore) (index sls.Index, err error) {
	preIndex, err := store.GetIndex()
	if err != nil && !IsExceptedErrors(err, []string{IndexConfigNotExist}) {
		return index, fmt.Errorf("While building index, GetIndex got an error: %#v.", err)
	}
	token := d.Get("token").(string)
	if indexType == FullText {
		indexLine := &sls.IndexLine{
			Token:         strings.Split(token, ""),
			CaseSensitive: d.Get("case_sensitive").(bool),
			Chn:           d.Get("include_chinese").(bool),
		}
		if preIndex == nil {
			return sls.Index{
				Line: indexLine,
			}, nil
		}
		preIndex.Line = indexLine
		index = *preIndex
		return
	}
	key := strings.Trim(d.Get("field_name").(string), " ")
	if key == "" {
		return index, fmt.Errorf("'field_name' is required when 'index_type' is '%s'.", Field)
	}
	value := sls.IndexKey{
		Type:          d.Get("field_type").(string),
		Alias:         d.Get("field_alias").(string),
		DocValue:      d.Get("enable_analytics").(bool),
		Token:         strings.Split(d.Get("token").(string), ""),
		CaseSensitive: d.Get("case_sensitive").(bool),
		Chn:           d.Get("include_chinese").(bool),
	}
	if value.Type == string(TextType) {
		if len(token) == 0 {
			return index, fmt.Errorf("'token' is required when 'field_type' is '%s'.", TextType)
		}
		value.Token = strings.Split(token, "")
	}
	if preIndex == nil {
		return sls.Index{
			Keys: map[string]sls.IndexKey{
				key: value,
			},
		}, nil
	}

	preIndex.Keys[key] = value
	index = *preIndex
	return
}
