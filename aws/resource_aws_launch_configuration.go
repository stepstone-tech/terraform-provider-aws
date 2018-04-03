package aws

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsLaunchConfiguration() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsLaunchConfigurationCreate,
		Read:   resourceAwsLaunchConfigurationRead,
		Delete: resourceAwsLaunchConfigurationDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		// CustomizeDiff: func(diff *schema.ResourceDiff, v interface{}) error {
		// 	if v, ok := diff.GetOk("block_device_mapping"); ok {
		// 		ebsDevice := v.(*schema.Set).List()
		// 		for _, device := range ebsDevice {
		// 			m := device.(map[string]interface{})
		// 			// TODO: Validate conflicting "virtual_name" & "ebs"
		// 			// TODO: Validate conflicting "ebs" && "no_device"
		//			// TODO: Validate conflicting "is_root_device" && "device_name"
		// 		}
		// 	}
		// 	return nil
		// },

		Schema: map[string]*schema.Schema{
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name_prefix"},
				ValidateFunc:  validation.StringLenBetween(1, 255),
			},

			"name_prefix": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringLenBetween(1, 255-resource.UniqueIDSuffixLength),
			},

			"image_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"instance_type": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"iam_instance_profile": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"key_name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"user_data": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				StateFunc: func(v interface{}) string {
					switch v.(type) {
					case string:
						hash := sha1.Sum([]byte(v.(string)))
						return hex.EncodeToString(hash[:])
					default:
						return ""
					}
				},
				ValidateFunc: validation.StringLenBetween(1, 16384),
			},

			"security_groups": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"vpc_classic_link_id": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"vpc_classic_link_security_groups": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"associate_public_ip_address": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Default:  false,
			},

			"spot_price": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"ebs_optimized": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"placement_tenancy": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"enable_monitoring": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Default:  true,
			},

			"block_device_mapping": {
				Type:          schema.TypeSet,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"ebs_block_device", "ephemeral_block_device", "root_block_device"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"device_name": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"virtual_name": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"no_device": {
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
						},

						"is_root_device": {
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
						},

						"ebs": {
							Type:     schema.TypeList,
							MaxItems: 1,
							Optional: true,
							ForceNew: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"delete_on_termination": {
										Type:     schema.TypeBool,
										Optional: true,
										Default:  true,
										ForceNew: true,
									},
									"iops": {
										Type:     schema.TypeInt,
										Optional: true,
										Computed: true,
										ForceNew: true,
									},
									"snapshot_id": {
										Type:     schema.TypeString,
										Optional: true,
										Computed: true,
										ForceNew: true,
									},
									"volume_size": {
										Type:     schema.TypeInt,
										Optional: true,
										Computed: true,
										ForceNew: true,
									},
									"volume_type": {
										Type:     schema.TypeString,
										Optional: true,
										Computed: true,
										ForceNew: true,
									},
									"encrypted": {
										Type:     schema.TypeBool,
										Optional: true,
										Computed: true,
										ForceNew: true,
									},
								},
							},
						},
					},
				},
			},

			// TODO: Deprecated fields, remove in the next major version
			"ebs_block_device": {
				Type:       schema.TypeSet,
				Optional:   true,
				Computed:   true,
				Deprecated: "Use 'block_device_mapping' instead.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"device_name": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},

						"delete_on_termination": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
							ForceNew: true,
						},

						"iops": {
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},

						"snapshot_id": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},

						"volume_size": {
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},

						"volume_type": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},

						"encrypted": {
							Type:     schema.TypeBool,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},
					},
				},
			},
			"ephemeral_block_device": {
				Type:       schema.TypeSet,
				Optional:   true,
				ForceNew:   true,
				Deprecated: "Use 'block_device_mapping' instead.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"device_name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"virtual_name": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				Set: func(v interface{}) int {
					var buf bytes.Buffer
					m := v.(map[string]interface{})
					buf.WriteString(fmt.Sprintf("%s-", m["device_name"].(string)))
					buf.WriteString(fmt.Sprintf("%s-", m["virtual_name"].(string)))
					return hashcode.String(buf.String())
				},
			},
			"root_block_device": {
				Type:       schema.TypeList,
				Optional:   true,
				Computed:   true,
				MaxItems:   1,
				Deprecated: "Use 'block_device_mapping' instead.",
				Elem: &schema.Resource{
					// "You can only modify the volume size, volume type, and Delete on
					// Termination flag on the block device mapping entry for the root
					// device volume." - bit.ly/ec2bdmap
					Schema: map[string]*schema.Schema{
						"delete_on_termination": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
							ForceNew: true,
						},

						"iops": {
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},

						"volume_size": {
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},

						"volume_type": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},
					},
				},
			},
		},
	}
}

func resourceAwsLaunchConfigurationCreate(d *schema.ResourceData, meta interface{}) error {
	autoscalingconn := meta.(*AWSClient).autoscalingconn
	ec2conn := meta.(*AWSClient).ec2conn

	createLaunchConfigurationOpts := autoscaling.CreateLaunchConfigurationInput{
		LaunchConfigurationName: aws.String(d.Get("name").(string)),
		ImageId:                 aws.String(d.Get("image_id").(string)),
		InstanceType:            aws.String(d.Get("instance_type").(string)),
		EbsOptimized:            aws.Bool(d.Get("ebs_optimized").(bool)),
	}

	if v, ok := d.GetOk("user_data"); ok {
		userData := base64Encode([]byte(v.(string)))
		createLaunchConfigurationOpts.UserData = aws.String(userData)
	}

	createLaunchConfigurationOpts.InstanceMonitoring = &autoscaling.InstanceMonitoring{
		Enabled: aws.Bool(d.Get("enable_monitoring").(bool)),
	}

	if v, ok := d.GetOk("iam_instance_profile"); ok {
		createLaunchConfigurationOpts.IamInstanceProfile = aws.String(v.(string))
	}

	if v, ok := d.GetOk("placement_tenancy"); ok {
		createLaunchConfigurationOpts.PlacementTenancy = aws.String(v.(string))
	}

	if v, ok := d.GetOk("associate_public_ip_address"); ok {
		createLaunchConfigurationOpts.AssociatePublicIpAddress = aws.Bool(v.(bool))
	}

	if v, ok := d.GetOk("key_name"); ok {
		createLaunchConfigurationOpts.KeyName = aws.String(v.(string))
	}
	if v, ok := d.GetOk("spot_price"); ok {
		createLaunchConfigurationOpts.SpotPrice = aws.String(v.(string))
	}

	if v, ok := d.GetOk("security_groups"); ok {
		createLaunchConfigurationOpts.SecurityGroups = expandStringList(
			v.(*schema.Set).List(),
		)
	}

	if v, ok := d.GetOk("vpc_classic_link_id"); ok {
		createLaunchConfigurationOpts.ClassicLinkVPCId = aws.String(v.(string))
	}

	if v, ok := d.GetOk("vpc_classic_link_security_groups"); ok {
		createLaunchConfigurationOpts.ClassicLinkVPCSecurityGroups = expandStringList(
			v.(*schema.Set).List(),
		)
	}

	if v, ok := d.GetOk("block_device_mapping"); ok {
		var err error
		amiId := d.Get("image_id").(string)
		createLaunchConfigurationOpts.BlockDeviceMappings, err = expandAutoscalingBlockDeviceMappings(v.([]interface{}), amiId, ec2conn)
		if err != nil {
			return err
		}
	} else {
		// TODO: Deprecated fields, remove in the next major version
		var blockDevices []*autoscaling.BlockDeviceMapping

		// We'll use this to detect if we're declaring it incorrectly as an ebs_block_device.
		rootDeviceName, err := fetchRootDeviceName(d.Get("image_id").(string), ec2conn)
		if err != nil {
			return err
		}
		if rootDeviceName == nil {
			// We do this so the value is empty so we don't have to do nil checks later
			var blank string
			rootDeviceName = &blank
		}

		if v, ok := d.GetOk("ebs_block_device"); ok {
			vL := v.(*schema.Set).List()
			for _, v := range vL {
				bd := v.(map[string]interface{})

				ebs := &autoscaling.Ebs{
					DeleteOnTermination: aws.Bool(bd["delete_on_termination"].(bool)),
				}

				if v, ok := bd["snapshot_id"].(string); ok && v != "" {
					ebs.SnapshotId = aws.String(v)
				}

				if v, ok := bd["encrypted"].(bool); ok && v {
					ebs.Encrypted = aws.Bool(v)
				}

				if v, ok := bd["volume_size"].(int); ok && v != 0 {
					ebs.VolumeSize = aws.Int64(int64(v))
				}

				if v, ok := bd["volume_type"].(string); ok && v != "" {
					ebs.VolumeType = aws.String(v)
				}

				if v, ok := bd["iops"].(int); ok && v > 0 {
					ebs.Iops = aws.Int64(int64(v))
				}

				if *aws.String(bd["device_name"].(string)) == *rootDeviceName {
					return fmt.Errorf("Root device (%s) declared as an 'ebs_block_device'.  Use 'root_block_device' keyword.", *rootDeviceName)
				}

				blockDevices = append(blockDevices, &autoscaling.BlockDeviceMapping{
					DeviceName: aws.String(bd["device_name"].(string)),
					Ebs:        ebs,
				})
			}
		}

		if v, ok := d.GetOk("ephemeral_block_device"); ok {
			vL := v.(*schema.Set).List()
			for _, v := range vL {
				bd := v.(map[string]interface{})
				blockDevices = append(blockDevices, &autoscaling.BlockDeviceMapping{
					DeviceName:  aws.String(bd["device_name"].(string)),
					VirtualName: aws.String(bd["virtual_name"].(string)),
				})
			}
		}

		if v, ok := d.GetOk("root_block_device"); ok {
			vL := v.([]interface{})
			for _, v := range vL {
				bd := v.(map[string]interface{})
				ebs := &autoscaling.Ebs{
					DeleteOnTermination: aws.Bool(bd["delete_on_termination"].(bool)),
				}

				if v, ok := bd["volume_size"].(int); ok && v != 0 {
					ebs.VolumeSize = aws.Int64(int64(v))
				}

				if v, ok := bd["volume_type"].(string); ok && v != "" {
					ebs.VolumeType = aws.String(v)
				}

				if v, ok := bd["iops"].(int); ok && v > 0 {
					ebs.Iops = aws.Int64(int64(v))
				}

				if dn, err := fetchRootDeviceName(d.Get("image_id").(string), ec2conn); err == nil {
					if dn == nil {
						return fmt.Errorf(
							"Expected to find a Root Device name for AMI (%s), but got none",
							d.Get("image_id").(string))
					}
					blockDevices = append(blockDevices, &autoscaling.BlockDeviceMapping{
						DeviceName: dn,
						Ebs:        ebs,
					})
				} else {
					return err
				}
			}
		}

		if len(blockDevices) > 0 {
			createLaunchConfigurationOpts.BlockDeviceMappings = blockDevices
		}
	}

	var lcName string
	if v, ok := d.GetOk("name"); ok {
		lcName = v.(string)
	} else if v, ok := d.GetOk("name_prefix"); ok {
		lcName = resource.PrefixedUniqueId(v.(string))
	} else {
		lcName = resource.UniqueId()
	}
	createLaunchConfigurationOpts.LaunchConfigurationName = aws.String(lcName)

	log.Printf("[DEBUG] autoscaling create launch configuration: %s", createLaunchConfigurationOpts)

	// IAM profiles can take ~10 seconds to propagate in AWS:
	// http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/iam-roles-for-amazon-ec2.html#launch-instance-with-role-console
	err := resource.Retry(90*time.Second, func() *resource.RetryError {
		_, err := autoscalingconn.CreateLaunchConfiguration(&createLaunchConfigurationOpts)
		if err != nil {
			if isAWSErr(err, "ValidationError", "Invalid IamInstanceProfile") {
				return resource.RetryableError(err)
			}
			if isAWSErr(err, "ValidationError", "You are not authorized to perform this operation") {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("Error creating launch configuration: %s", err)
	}

	d.SetId(lcName)
	log.Printf("[INFO] launch configuration ID: %s", d.Id())

	// We put a Retry here since sometimes eventual consistency bites
	// us and we need to retry a few times to get the LC to load properly
	return resource.Retry(30*time.Second, func() *resource.RetryError {
		err := resourceAwsLaunchConfigurationRead(d, meta)
		if err != nil {
			return resource.RetryableError(err)
		}
		return nil
	})
}

func resourceAwsLaunchConfigurationRead(d *schema.ResourceData, meta interface{}) error {
	autoscalingconn := meta.(*AWSClient).autoscalingconn
	ec2conn := meta.(*AWSClient).ec2conn

	describeOpts := autoscaling.DescribeLaunchConfigurationsInput{
		LaunchConfigurationNames: []*string{aws.String(d.Id())},
	}

	log.Printf("[DEBUG] launch configuration describe configuration: %s", describeOpts)
	describConfs, err := autoscalingconn.DescribeLaunchConfigurations(&describeOpts)
	if err != nil {
		return fmt.Errorf("Error retrieving launch configuration: %s", err)
	}
	if len(describConfs.LaunchConfigurations) == 0 {
		log.Printf("[WARN] Launch Configuration (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	// Verify AWS returned our launch configuration
	if *describConfs.LaunchConfigurations[0].LaunchConfigurationName != d.Id() {
		return fmt.Errorf(
			"Unable to find launch configuration: %#v",
			describConfs.LaunchConfigurations)
	}

	lc := describConfs.LaunchConfigurations[0]

	d.Set("key_name", lc.KeyName)
	d.Set("image_id", lc.ImageId)
	d.Set("instance_type", lc.InstanceType)
	d.Set("name", lc.LaunchConfigurationName)

	d.Set("iam_instance_profile", lc.IamInstanceProfile)
	d.Set("ebs_optimized", lc.EbsOptimized)
	d.Set("spot_price", lc.SpotPrice)
	d.Set("enable_monitoring", lc.InstanceMonitoring.Enabled)
	d.Set("security_groups", lc.SecurityGroups)
	d.Set("associate_public_ip_address", lc.AssociatePublicIpAddress)

	d.Set("vpc_classic_link_id", lc.ClassicLinkVPCId)
	d.Set("vpc_classic_link_security_groups", lc.ClassicLinkVPCSecurityGroups)

	bdms, err := flattenAutoscalingBlockDeviceMappings(lc.BlockDeviceMappings, *lc.ImageId, ec2conn)
	if err != nil {
		return err
	}
	err = d.Set("block_device_mapping", bdms)
	if err != nil {
		return err
	}

	if err := readLCBlockDevices(d, lc, ec2conn); err != nil {
		return err
	}

	return nil
}

func resourceAwsLaunchConfigurationDelete(d *schema.ResourceData, meta interface{}) error {
	autoscalingconn := meta.(*AWSClient).autoscalingconn

	log.Printf("[DEBUG] Launch Configuration destroy: %v", d.Id())
	_, err := autoscalingconn.DeleteLaunchConfiguration(
		&autoscaling.DeleteLaunchConfigurationInput{
			LaunchConfigurationName: aws.String(d.Id()),
		})
	if err != nil {
		if isAWSErr(err, "InvalidConfiguration.NotFound", "") {
			log.Printf("[DEBUG] Launch configuration (%s) not found", d.Id())
			return nil
		}

		return err
	}

	return nil
}

func readLCBlockDevices(d *schema.ResourceData, lc *autoscaling.LaunchConfiguration, ec2conn *ec2.EC2) error {
	ibds, err := readBlockDevicesFromLaunchConfiguration(d, lc, ec2conn)
	if err != nil {
		return err
	}

	if err := d.Set("ebs_block_device", ibds["ebs"]); err != nil {
		return err
	}
	if err := d.Set("ephemeral_block_device", ibds["ephemeral"]); err != nil {
		return err
	}
	if ibds["root"] != nil {
		if err := d.Set("root_block_device", []interface{}{ibds["root"]}); err != nil {
			return err
		}
	} else {
		d.Set("root_block_device", []interface{}{})
	}

	return nil
}

func readBlockDevicesFromLaunchConfiguration(d *schema.ResourceData, lc *autoscaling.LaunchConfiguration, ec2conn *ec2.EC2) (
	map[string]interface{}, error) {
	blockDevices := make(map[string]interface{})
	blockDevices["ebs"] = make([]map[string]interface{}, 0)
	blockDevices["ephemeral"] = make([]map[string]interface{}, 0)
	blockDevices["root"] = nil
	if len(lc.BlockDeviceMappings) == 0 {
		return nil, nil
	}
	rootDeviceName, err := fetchRootDeviceName(d.Get("image_id").(string), ec2conn)
	if err != nil {
		return nil, err
	}
	if rootDeviceName == nil {
		// We do this so the value is empty so we don't have to do nil checks later
		var blank string
		rootDeviceName = &blank
	}
	for _, bdm := range lc.BlockDeviceMappings {
		bd := make(map[string]interface{})
		if bdm.Ebs != nil && bdm.Ebs.DeleteOnTermination != nil {
			bd["delete_on_termination"] = *bdm.Ebs.DeleteOnTermination
		}
		if bdm.Ebs != nil && bdm.Ebs.VolumeSize != nil {
			bd["volume_size"] = *bdm.Ebs.VolumeSize
		}
		if bdm.Ebs != nil && bdm.Ebs.VolumeType != nil {
			bd["volume_type"] = *bdm.Ebs.VolumeType
		}
		if bdm.Ebs != nil && bdm.Ebs.Iops != nil {
			bd["iops"] = *bdm.Ebs.Iops
		}

		if bdm.DeviceName != nil && *bdm.DeviceName == *rootDeviceName {
			blockDevices["root"] = bd
		} else {
			if bdm.Ebs != nil && bdm.Ebs.Encrypted != nil {
				bd["encrypted"] = *bdm.Ebs.Encrypted
			}
			if bdm.DeviceName != nil {
				bd["device_name"] = *bdm.DeviceName
			}
			if bdm.VirtualName != nil {
				bd["virtual_name"] = *bdm.VirtualName
				blockDevices["ephemeral"] = append(blockDevices["ephemeral"].([]map[string]interface{}), bd)
			} else {
				if bdm.Ebs != nil && bdm.Ebs.SnapshotId != nil {
					bd["snapshot_id"] = *bdm.Ebs.SnapshotId
				}
				blockDevices["ebs"] = append(blockDevices["ebs"].([]map[string]interface{}), bd)
			}
		}
	}
	return blockDevices, nil
}

func expandAutoscalingBlockDeviceMappings(in []interface{}, amiId string, ec2conn *ec2.EC2) ([]*autoscaling.BlockDeviceMapping, error) {
	if len(in) == 0 || in[0] == nil {
		return nil, nil
	}

	out := make([]*autoscaling.BlockDeviceMapping, len(in), len(in))
	for i, bdm := range in {
		m := bdm.(map[string]interface{})

		if v, ok := m["is_root_device"]; ok {
			isRoot := v.(bool)
			if isRoot {
				var err error
				out[i].DeviceName, err = fetchRootDeviceName(amiId, ec2conn)
				if err != nil {
					return nil, err
				}
			}
		}
		if v, ok := m["device_name"]; ok {
			out[i].DeviceName = aws.String(v.(string))
		}
		if v, ok := m["ebs"]; ok {
			out[i].Ebs = expandAutoscalingEbs(v.([]interface{}))
		}
		if v, ok := m["no_device"]; ok {
			out[i].NoDevice = aws.Bool(v.(bool))
		}
		if v, ok := m["virtual_name"]; ok {
			out[i].VirtualName = aws.String(v.(string))
		}
	}
	return out, nil
}

func expandAutoscalingEbs(in []interface{}) *autoscaling.Ebs {
	if len(in) == 0 || in[0] == nil {
		return nil
	}
	m := in[0].(map[string]interface{})

	ebs := &autoscaling.Ebs{
		DeleteOnTermination: aws.Bool(m["delete_on_termination"].(bool)),
	}

	if v, ok := m["snapshot_id"].(string); ok && v != "" {
		ebs.SnapshotId = aws.String(v)
	}

	if v, ok := m["encrypted"].(bool); ok && v {
		ebs.Encrypted = aws.Bool(v)
	}

	if v, ok := m["volume_size"].(int); ok && v != 0 {
		ebs.VolumeSize = aws.Int64(int64(v))
	}

	if v, ok := m["volume_type"].(string); ok && v != "" {
		ebs.VolumeType = aws.String(v)
	}

	if v, ok := m["iops"].(int); ok && v > 0 {
		ebs.Iops = aws.Int64(int64(v))
	}

	return ebs
}

func flattenAutoscalingBlockDeviceMappings(in []*autoscaling.BlockDeviceMapping, amiId string, ec2conn *ec2.EC2) ([]interface{}, error) {
	if len(in) == 0 {
		return []interface{}{}, nil
	}

	out := make([]interface{}, len(in), len(in))
	for i, bdm := range in {
		m := make(map[string]interface{}, 0)
		if bdm.DeviceName != nil {
			m["device_name"] = *bdm.DeviceName

			rootDeviceName, err := fetchRootDeviceName(amiId, ec2conn)
			if err != nil {
				return nil, err
			}

			if *rootDeviceName == *bdm.DeviceName {
				m["is_root_device"] = true
			} else {
				m["is_root_device"] = false
			}
		}
		if bdm.Ebs != nil {
			m["ebs"] = flattenAutoscalingEbs(bdm.Ebs)
		}
		if bdm.NoDevice != nil {
			m["no_device"] = *bdm.NoDevice
		}
		if bdm.VirtualName != nil {
			m["virtual_name"] = *bdm.VirtualName
		}

		out[i] = m
	}

	return out, nil
}

func flattenAutoscalingEbs(in *autoscaling.Ebs) []interface{} {
	if in == nil {
		return []interface{}{}
	}

	m := make(map[string]interface{}, 0)
	if in.DeleteOnTermination != nil {
		m["delete_on_termination"] = *in.DeleteOnTermination
	}
	if in.Encrypted != nil {
		m["encrypted"] = *in.Encrypted
	}
	if in.Iops != nil {
		m["iops"] = *in.Iops
	}
	if in.SnapshotId != nil {
		m["snapshot_id"] = *in.SnapshotId
	}
	if in.VolumeSize != nil {
		m["volume_size"] = *in.VolumeSize
	}
	if in.VolumeType != nil {
		m["volume_type"] = *in.VolumeType
	}

	return []interface{}{m}
}
